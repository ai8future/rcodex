import { NextResponse } from 'next/server'
import fs from 'fs'
import path from 'path'
import os from 'os'

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

function readSchedules(): SchedulesData {
  if (!fs.existsSync(SCHEDULES_FILE)) {
    return { schedules: [] }
  }
  const content = fs.readFileSync(SCHEDULES_FILE, 'utf-8')
  return JSON.parse(content)
}

function writeSchedules(data: SchedulesData) {
  fs.writeFileSync(SCHEDULES_FILE, JSON.stringify(data, null, 2))
}

export async function PATCH(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params
    const body = await request.json()
    const data = readSchedules()

    const index = data.schedules.findIndex(s => s.id === id)
    if (index === -1) {
      return NextResponse.json({ error: 'Schedule not found' }, { status: 404 })
    }

    // Update allowed fields
    if (typeof body.enabled === 'boolean') {
      data.schedules[index].enabled = body.enabled
    }
    if (body.cron) {
      data.schedules[index].cron = body.cron
    }
    if (body.task) {
      data.schedules[index].task = body.task
    }

    writeSchedules(data)
    return NextResponse.json(data.schedules[index])
  } catch (error) {
    console.error('Error updating schedule:', error)
    return NextResponse.json({ error: 'Failed to update schedule' }, { status: 500 })
  }
}

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params
    const data = readSchedules()

    const index = data.schedules.findIndex(s => s.id === id)
    if (index === -1) {
      return NextResponse.json({ error: 'Schedule not found' }, { status: 404 })
    }

    data.schedules.splice(index, 1)
    writeSchedules(data)

    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting schedule:', error)
    return NextResponse.json({ error: 'Failed to delete schedule' }, { status: 500 })
  }
}
