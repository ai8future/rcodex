'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'

interface Schedule {
  id: string
  repo: string
  task: string
  cron: string
  enabled: boolean
  created: string
}

interface RepoOption {
  name: string
  path: string
}

const TASKS = ['suite', 'audit', 'test', 'fix', 'refactor', 'quick', 'grade']

const CRON_PRESETS = [
  { label: 'Every day at 9am', value: '0 9 * * *' },
  { label: 'Every Monday at 9am', value: '0 9 * * 1' },
  { label: 'Every Sunday at midnight', value: '0 0 * * 0' },
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Custom', value: '' }
]

function describeCron(cron: string): string {
  const preset = CRON_PRESETS.find(p => p.value === cron)
  if (preset && preset.label !== 'Custom') return preset.label

  // Simple cron description
  const parts = cron.split(' ')
  if (parts.length !== 5) return cron

  const [min, hour, , , dow] = parts

  if (dow !== '*') {
    const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
    const dayName = days[parseInt(dow)] || dow
    return `${dayName} at ${hour}:${min.padStart(2, '0')}`
  }

  if (hour !== '*') {
    return `Daily at ${hour}:${min.padStart(2, '0')}`
  }

  return cron
}

export default function SchedulesPage() {
  const [schedules, setSchedules] = useState<Schedule[]>([])
  const [repos, setRepos] = useState<RepoOption[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)

  // Form state
  const [selectedRepo, setSelectedRepo] = useState('')
  const [selectedTask, setSelectedTask] = useState('suite')
  const [selectedCronPreset, setSelectedCronPreset] = useState('0 9 * * 1')
  const [customCron, setCustomCron] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    Promise.all([
      fetch('/api/schedules').then(r => r.json()),
      fetch('/api/repos').then(r => r.json())
    ]).then(([schedulesData, reposData]) => {
      setSchedules(Array.isArray(schedulesData) ? schedulesData : [])
      setRepos(reposData.map((r: { name: string; path: string }) => ({ name: r.name, path: r.path })))
      if (reposData.length > 0) {
        setSelectedRepo(reposData[0].path)
      }
      setLoading(false)
    }).catch(err => {
      console.error('Error loading data:', err)
      setLoading(false)
    })
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)

    const cron = selectedCronPreset || customCron

    try {
      const res = await fetch('/api/schedules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repo: selectedRepo,
          task: selectedTask,
          cron
        })
      })

      if (res.ok) {
        const newSchedule = await res.json()
        setSchedules([...schedules, newSchedule])
        setShowForm(false)
        setCustomCron('')
      }
    } catch (err) {
      console.error('Error creating schedule:', err)
    }

    setSubmitting(false)
  }

  const toggleSchedule = async (id: string, enabled: boolean) => {
    try {
      const res = await fetch(`/api/schedules/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !enabled })
      })

      if (res.ok) {
        setSchedules(schedules.map(s =>
          s.id === id ? { ...s, enabled: !enabled } : s
        ))
      }
    } catch (err) {
      console.error('Error toggling schedule:', err)
    }
  }

  const deleteSchedule = async (id: string) => {
    if (!confirm('Delete this schedule?')) return

    try {
      const res = await fetch(`/api/schedules/${id}`, { method: 'DELETE' })
      if (res.ok) {
        setSchedules(schedules.filter(s => s.id !== id))
      }
    } catch (err) {
      console.error('Error deleting schedule:', err)
    }
  }

  const getRepoName = (repoPath: string) => {
    return repoPath.split('/').pop() || repoPath
  }

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-950">
      <header className="border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm">
            <Link href="/" className="text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100">
              Dashboard
            </Link>
            <span className="text-zinc-400">/</span>
            <span className="text-zinc-900 dark:text-zinc-100">Schedules</span>
          </div>
          <button
            onClick={() => setShowForm(true)}
            className="px-4 py-2 bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 rounded-lg text-sm font-medium hover:bg-zinc-800 dark:hover:bg-zinc-200"
          >
            Add Schedule
          </button>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 py-8">
        {loading ? (
          <div className="text-center py-12 text-zinc-500">Loading...</div>
        ) : (
          <>
            {showForm && (
              <div className="mb-8 bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
                <h2 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-4">
                  New Schedule
                </h2>
                <form onSubmit={handleSubmit} className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1">
                      Repository
                    </label>
                    <select
                      value={selectedRepo}
                      onChange={e => setSelectedRepo(e.target.value)}
                      className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100"
                    >
                      {repos.map(repo => (
                        <option key={repo.path} value={repo.path}>
                          {repo.name}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1">
                      Task
                    </label>
                    <select
                      value={selectedTask}
                      onChange={e => setSelectedTask(e.target.value)}
                      className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100"
                    >
                      {TASKS.map(task => (
                        <option key={task} value={task}>
                          {task}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1">
                      Schedule
                    </label>
                    <select
                      value={selectedCronPreset}
                      onChange={e => setSelectedCronPreset(e.target.value)}
                      className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100"
                    >
                      {CRON_PRESETS.map(preset => (
                        <option key={preset.value} value={preset.value}>
                          {preset.label}
                        </option>
                      ))}
                    </select>
                  </div>

                  {selectedCronPreset === '' && (
                    <div>
                      <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1">
                        Custom Cron Expression
                      </label>
                      <input
                        type="text"
                        value={customCron}
                        onChange={e => setCustomCron(e.target.value)}
                        placeholder="0 9 * * 1"
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100"
                      />
                      <p className="mt-1 text-xs text-zinc-500">
                        Format: minute hour day month weekday
                      </p>
                    </div>
                  )}

                  <div className="flex gap-3">
                    <button
                      type="submit"
                      disabled={submitting}
                      className="px-4 py-2 bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 rounded-lg text-sm font-medium hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50"
                    >
                      {submitting ? 'Creating...' : 'Create Schedule'}
                    </button>
                    <button
                      type="button"
                      onClick={() => setShowForm(false)}
                      className="px-4 py-2 border border-zinc-300 dark:border-zinc-700 rounded-lg text-sm font-medium text-zinc-700 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800"
                    >
                      Cancel
                    </button>
                  </div>
                </form>
              </div>
            )}

            {schedules.length === 0 ? (
              <div className="text-center py-12 text-zinc-500">
                No schedules configured
              </div>
            ) : (
              <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/50">
                      <th className="text-left px-6 py-3 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                        Repo
                      </th>
                      <th className="text-left px-6 py-3 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                        Task
                      </th>
                      <th className="text-left px-6 py-3 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                        Schedule
                      </th>
                      <th className="text-left px-6 py-3 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                        Status
                      </th>
                      <th className="text-right px-6 py-3 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {schedules.map(schedule => (
                      <tr
                        key={schedule.id}
                        className="border-b border-zinc-100 dark:border-zinc-800 last:border-0"
                      >
                        <td className="px-6 py-4 text-zinc-900 dark:text-zinc-100 font-medium">
                          {getRepoName(schedule.repo)}
                        </td>
                        <td className="px-6 py-4 text-zinc-600 dark:text-zinc-400">
                          {schedule.task}
                        </td>
                        <td className="px-6 py-4 text-zinc-600 dark:text-zinc-400">
                          {describeCron(schedule.cron)}
                        </td>
                        <td className="px-6 py-4">
                          <button
                            onClick={() => toggleSchedule(schedule.id, schedule.enabled)}
                            className={`px-2 py-1 rounded text-xs font-medium ${
                              schedule.enabled
                                ? 'bg-green-100 text-green-800'
                                : 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400'
                            }`}
                          >
                            {schedule.enabled ? 'Enabled' : 'Disabled'}
                          </button>
                        </td>
                        <td className="px-6 py-4 text-right">
                          <button
                            onClick={() => deleteSchedule(schedule.id)}
                            className="text-red-600 hover:text-red-800 text-sm"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )}
      </main>
    </div>
  )
}
