import { useState } from 'react'

export default function Home() {
  const [q, setQ] = useState('json')
  const [results, setResults] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const gateway = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080'

  async function onSearch(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    try {
      const res = await fetch(`${gateway}/api/search?q=${encodeURIComponent(q)}`)
      const data = await res.json()
      setResults(data.results || [])
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{padding: 24, fontFamily: 'Inter, system-ui, sans-serif'}}>
      <h1>PolyScale</h1>
      <form onSubmit={onSearch}>
        <input value={q} onChange={e => setQ(e.target.value)} placeholder="Search code..." style={{width: 400, padding: 8}}/>
        <button type="submit" style={{marginLeft: 8, padding: '8px 12px'}}>Search</button>
      </form>
      {loading && <p>Searching...</p>}
      <ul>
        {results.map((r, i) => (
          <li key={i} style={{marginTop: 12}}>
            <div><strong>{r.repo}</strong> — {r.path} — <em>{r.language}</em> — score {r.score?.toFixed?.(2)}</div>
            <pre style={{background: '#f5f5f5', padding: 12, whiteSpace: 'pre-wrap'}}>{r.snippet}</pre>
          </li>
        ))}
      </ul>
    </div>
  )
}


