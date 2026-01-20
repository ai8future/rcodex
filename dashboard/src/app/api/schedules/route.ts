import { NextResponse } from 'next/server'
import fs from 'fs'
import path from 'path'
import os from 'os'
import crypto from 'crypto'

const SCHEDULES_FILE = path.join(os.homedir(), '.rcodegen/schedules.json')

interface Schedule {
  id: string
  repo: string
  task: string
  cron: string
  enabled: boolean
  created: string
}

interface SchedulesData {
  schedules: Schedule[]
}

function ensureDir() {
  const dir = path.dirname(SCHEDULES_FILE)
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true })
  }
}

function readSchedules(): SchedulesData {
  if (!fs.existsSync(SCHEDULES_FILE)) {
    return { schedules: [] }
  }
  const content = fs.readFileSync(SCHEDULES_FILE, 'utf-8')
  return JSON.parse(content)
}

function writeSchedules(data: SchedulesData) {
  ensureDir()
  fs.writeFileSync(SCHEDULES_FILE, JSON.stringify(data, null, 2))
}

export async function GET() {
  try {
    const data = readSchedules()
    return NextResponse.json(data.schedules)
  } catch (error) {
    console.error('Error reading schedules:', error)
    return NextResponse.json({ error: 'Failed to read schedules' }, { status: 500 })
  }
}

export async function POST(request: Request) {
  try {
    const body = await request.json()
    const { repo, task, cron } = body

    if (!repo || !task || !cron) {
      return NextResponse.json({ error: 'Missing required fields' }, { status: 400 })
    }

    const data = readSchedules()
    const schedule: Schedule = {
      id: crypto.randomUUID(),
      repo,
      task,
      cron,
      enabled: true,
      created: new Date().toISOString()
    }

    data.schedules.push(schedule)
    writeSchedules(data)

    return NextResponse.json(schedule, { status: 201 })
  } catch (error) {
    console.error('Error creating schedule:', error)
    return NextResponse.json({ error: 'Failed to create schedule' }, { status: 500 })
  }
}
