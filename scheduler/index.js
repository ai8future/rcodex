import { CronJob } from 'cron'
import fs from 'fs'
import path from 'path'
import os from 'os'
import { spawn } from 'child_process'

const SCHEDULES_FILE = path.join(os.homedir(), '.rcodegen/schedules.json')
const STATUS_FILE = path.join(os.homedir(), '.rcodegen/scheduler-status.json')
const POLL_INTERVAL = 60000 // 60 seconds

const activeJobs = new Map()
let recentRuns = []
const MAX_RECENT_RUNS = 20

function log(msg) {
  const timestamp = new Date().toISOString()
  console.log(`[${timestamp}] ${msg}`)
}

function ensureDir(filePath) {
  const dir = path.dirname(filePath)
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true })
  }
}

function readSchedules() {
  if (!fs.existsSync(SCHEDULES_FILE)) {
    return []
  }
  try {
    const content = fs.readFileSync(SCHEDULES_FILE, 'utf-8')
    const data = JSON.parse(content)
    return data.schedules || []
  } catch (err) {
    log(`Error reading schedules: ${err.message}`)
    return []
  }
}

function writeStatus() {
  ensureDir(STATUS_FILE)
  const status = {
    pid: process.pid,
    started: startTime.toISOString(),
    lastHeartbeat: new Date().toISOString(),
    recentRuns
  }
  fs.writeFileSync(STATUS_FILE, JSON.stringify(status, null, 2))
}

function runTask(schedule) {
  const repoName = path.basename(schedule.repo)
  log(`Starting task: ${schedule.task} on ${repoName}`)

  const run = {
    id: schedule.id,
    repo: repoName,
    task: schedule.task,
    startedAt: new Date().toISOString(),
    finishedAt: null,
    exitCode: null
  }

  // Spawn rcodex process
  const proc = spawn('rcodex', [schedule.task], {
    cwd: schedule.repo,
    stdio: 'inherit'
  })

  proc.on('error', (err) => {
    log(`Error running task on ${repoName}: ${err.message}`)
    run.finishedAt = new Date().toISOString()
    run.exitCode = 1
    addRecentRun(run)
  })

  proc.on('close', (code) => {
    log(`Task ${schedule.task} on ${repoName} finished with code ${code}`)
    run.finishedAt = new Date().toISOString()
    run.exitCode = code
    addRecentRun(run)
  })
}

function addRecentRun(run) {
  recentRuns.unshift(run)
  if (recentRuns.length > MAX_RECENT_RUNS) {
    recentRuns = recentRuns.slice(0, MAX_RECENT_RUNS)
  }
  writeStatus()
}

function createJob(schedule) {
  try {
    const job = new CronJob(
      schedule.cron,
      () => runTask(schedule),
      null,
      true, // start immediately
      'America/New_York' // timezone
    )
    return job
  } catch (err) {
    log(`Invalid cron expression for ${schedule.id}: ${schedule.cron}`)
    return null
  }
}

function syncJobs() {
  const schedules = readSchedules()
  const currentIds = new Set(schedules.filter(s => s.enabled).map(s => s.id))

  // Stop jobs that are no longer in the schedule or disabled
  for (const [id, job] of activeJobs) {
    if (!currentIds.has(id)) {
      log(`Stopping job ${id}`)
      job.stop()
      activeJobs.delete(id)
    }
  }

  // Start new jobs
  for (const schedule of schedules) {
    if (!schedule.enabled) continue
    if (activeJobs.has(schedule.id)) continue

    const job = createJob(schedule)
    if (job) {
      log(`Starting job ${schedule.id}: ${schedule.task} on ${path.basename(schedule.repo)} (${schedule.cron})`)
      activeJobs.set(schedule.id, job)
    }
  }

  writeStatus()
}

// Main
const startTime = new Date()
log('Scheduler starting...')
log(`PID: ${process.pid}`)
log(`Schedules file: ${SCHEDULES_FILE}`)
log(`Status file: ${STATUS_FILE}`)

// Initial sync
syncJobs()

// Poll for changes
setInterval(syncJobs, POLL_INTERVAL)

// Heartbeat
setInterval(writeStatus, 30000)

// Handle shutdown
process.on('SIGINT', () => {
  log('Shutting down...')
  for (const [id, job] of activeJobs) {
    job.stop()
  }
  // Clear status file on clean shutdown
  if (fs.existsSync(STATUS_FILE)) {
    const status = JSON.parse(fs.readFileSync(STATUS_FILE, 'utf-8'))
    status.lastHeartbeat = new Date(0).toISOString() // Set to epoch to indicate stopped
    fs.writeFileSync(STATUS_FILE, JSON.stringify(status, null, 2))
  }
  process.exit(0)
})

log('Scheduler running. Press Ctrl+C to stop.')
