import { NextResponse } from 'next/server'
import fs from 'fs'
import path from 'path'
import os from 'os'

const STATUS_FILE = path.join(os.homedir(), '.rcodegen/scheduler-status.json')

interface SchedulerStatus {
  pid: number
  started: string
  lastHeartbeat: string
  recentRuns: Array<{
    id: string
    repo: string
    task: string
    startedAt: string
    finishedAt: string
    exitCode: number
  }>
}

export async function GET() {
  try {
    if (!fs.existsSync(STATUS_FILE)) {
      return NextResponse.json({
        running: false,
        lastHeartbeat: null,
        recentRuns: []
      })
    }

    const content = fs.readFileSync(STATUS_FILE, 'utf-8')
    const status: SchedulerStatus = JSON.parse(content)

    // Check if daemon is running (heartbeat within last 2 minutes)
    const lastHeartbeat = new Date(status.lastHeartbeat)
    const now = new Date()
    const diffMs = now.getTime() - lastHeartbeat.getTime()
    const running = diffMs < 2 * 60 * 1000

    return NextResponse.json({
      running,
      pid: status.pid,
      started: status.started,
      lastHeartbeat: status.lastHeartbeat,
      recentRuns: status.recentRuns || []
    })
  } catch (error) {
    console.error('Error reading daemon status:', error)
    return NextResponse.json({
      running: false,
      lastHeartbeat: null,
      recentRuns: []
    })
  }
}
