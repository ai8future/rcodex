'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'

interface ReportDetail {
  filename: string
  tool: string
  task: string
  date: string
  grade: number | null
  size: number
}

interface RepoData {
  name: string
  path: string
  reports: ReportDetail[]
  grouped: Record<string, ReportDetail[]>
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function GradeBadge({ grade }: { grade: number | null }) {
  if (grade === null) return null

  let colorClass = 'bg-red-100 text-red-800'
  if (grade >= 90) colorClass = 'bg-green-100 text-green-800'
  else if (grade >= 80) colorClass = 'bg-blue-100 text-blue-800'
  else if (grade >= 70) colorClass = 'bg-yellow-100 text-yellow-800'
  else if (grade >= 60) colorClass = 'bg-orange-100 text-orange-800'

  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${colorClass}`}>
      {grade.toFixed(0)}
    </span>
  )
}

function ToolBadge({ tool }: { tool: string }) {
  const colors: Record<string, string> = {
    codex: 'bg-purple-100 text-purple-800',
    gemini: 'bg-blue-100 text-blue-800',
    claude: 'bg-orange-100 text-orange-800'
  }
  const colorClass = colors[tool.toLowerCase()] || 'bg-zinc-100 text-zinc-800'

  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${colorClass}`}>
      {tool}
    </span>
  )
}

const TASK_ORDER = ['audit', 'test', 'fix', 'refactor', 'quick', 'grade']

export default function RepoPage() {
  const params = useParams()
  const name = params.name as string
  const [data, setData] = useState<RepoData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch(`/api/repos/${encodeURIComponent(name)}`)
      .then(res => res.json())
      .then(data => {
        if (data.error) {
          setError(data.error)
        } else {
          setData(data)
        }
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [name])

  const sortedTasks = data
    ? Object.keys(data.grouped).sort((a, b) => {
        const aIndex = TASK_ORDER.indexOf(a)
        const bIndex = TASK_ORDER.indexOf(b)
        if (aIndex === -1 && bIndex === -1) return a.localeCompare(b)
        if (aIndex === -1) return 1
        if (bIndex === -1) return -1
        return aIndex - bIndex
      })
    : []

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-950">
      <header className="border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
        <div className="max-w-6xl mx-auto px-6 py-4">
          <div className="flex items-center gap-2 text-sm text-zinc-500 dark:text-zinc-400 mb-1">
            <Link href="/" className="hover:text-zinc-900 dark:hover:text-zinc-100">
              Dashboard
            </Link>
            <span>/</span>
            <span className="text-zinc-900 dark:text-zinc-100">{decodeURIComponent(name)}</span>
          </div>
          <h1 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
            {decodeURIComponent(name)}
          </h1>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 py-8">
        {loading ? (
          <div className="text-center py-12 text-zinc-500">Loading...</div>
        ) : error ? (
          <div className="text-center py-12 text-red-500">{error}</div>
        ) : !data ? (
          <div className="text-center py-12 text-zinc-500">No data</div>
        ) : (
          <div className="space-y-8">
            {sortedTasks.map(task => (
              <div key={task}>
                <h2 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-3 capitalize">
                  {task}
                </h2>
                <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/50">
                        <th className="text-left px-4 py-2 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                          Date
                        </th>
                        <th className="text-left px-4 py-2 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                          Tool
                        </th>
                        <th className="text-left px-4 py-2 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                          Grade
                        </th>
                        <th className="text-left px-4 py-2 text-sm font-medium text-zinc-600 dark:text-zinc-400">
                          Size
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.grouped[task].map(report => (
                        <tr
                          key={report.filename}
                          className="border-b border-zinc-100 dark:border-zinc-800 last:border-0 hover:bg-zinc-50 dark:hover:bg-zinc-800/30"
                        >
                          <td className="px-4 py-3">
                            <Link
                              href={`/repo/${encodeURIComponent(name)}/report/${encodeURIComponent(report.filename)}`}
                              className="text-zinc-900 dark:text-zinc-100 hover:text-blue-600 dark:hover:text-blue-400"
                            >
                              {formatDate(report.date)}
                            </Link>
                          </td>
                          <td className="px-4 py-3">
                            <ToolBadge tool={report.tool} />
                          </td>
                          <td className="px-4 py-3">
                            <GradeBadge grade={report.grade} />
                          </td>
                          <td className="px-4 py-3 text-sm text-zinc-500 dark:text-zinc-400">
                            {formatSize(report.size)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
