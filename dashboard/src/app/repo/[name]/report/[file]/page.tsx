'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

export default function ReportPage() {
  const params = useParams()
  const name = params.name as string
  const file = params.file as string
  const [content, setContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch(`/api/repos/${encodeURIComponent(name)}/reports/${encodeURIComponent(file)}`)
      .then(res => res.json())
      .then(data => {
        if (data.error) {
          setError(data.error)
        } else {
          setContent(data.content)
        }
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [name, file])

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-950">
      <header className="border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 sticky top-0 z-10">
        <div className="max-w-6xl mx-auto px-6 py-4">
          <div className="flex items-center gap-2 text-sm text-zinc-500 dark:text-zinc-400 mb-1">
            <Link href="/" className="hover:text-zinc-900 dark:hover:text-zinc-100">
              Dashboard
            </Link>
            <span>/</span>
            <Link
              href={`/repo/${encodeURIComponent(name)}`}
              className="hover:text-zinc-900 dark:hover:text-zinc-100"
            >
              {decodeURIComponent(name)}
            </Link>
            <span>/</span>
            <span className="text-zinc-900 dark:text-zinc-100 truncate max-w-xs">
              {decodeURIComponent(file)}
            </span>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-6 py-8">
        {loading ? (
          <div className="text-center py-12 text-zinc-500">Loading...</div>
        ) : error ? (
          <div className="text-center py-12 text-red-500">{error}</div>
        ) : !content ? (
          <div className="text-center py-12 text-zinc-500">No content</div>
        ) : (
          <article className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 p-8 prose prose-zinc dark:prose-invert max-w-none prose-pre:bg-zinc-100 dark:prose-pre:bg-zinc-800 prose-pre:text-sm prose-code:text-sm prose-code:before:content-none prose-code:after:content-none">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
          </article>
        )}
      </main>
    </div>
  )
}
