import { useCallback, useRef, useState } from 'react'

interface FileUploadProps {
  /** Called with the selected file. */
  onFile: (file: File) => void
  /** Whether the database is currently being loaded. */
  loading: boolean
  /** Error message to display, or null. */
  error: string | null
  /** Optional callback shown as a dismiss button, e.g. when data is loaded. */
  onCancel?: () => void
}

/**
 * Full-screen drop zone presented before a database has been loaded.
 */
export default function FileUpload({ onFile, loading, error, onCancel }: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)

  const handleFiles = useCallback(
    (files: FileList | null) => {
      if (files && files.length > 0) {
        onFile(files[0])
      }
    },
    [onFile]
  )

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f4f5f6',
        zIndex: 10
      }}
      onDragOver={(e) => {
        e.preventDefault()
        setDragging(true)
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={(e) => {
        e.preventDefault()
        setDragging(false)
        handleFiles(e.dataTransfer.files)
      }}
    >
      <div
        style={{
          width: 420,
          padding: 40,
          textAlign: 'center',
          background: 'white',
          border: `2px dashed ${dragging ? '#2196f3' : '#c4c8cc'}`,
          borderRadius: 8,
          boxShadow: '0 2px 12px rgba(0,0,0,0.08)'
        }}
      >
        <h1 style={{ margin: '0 0 8px', fontSize: 20 }}>FogDB Viewer</h1>
        <p style={{ margin: '0 0 24px', color: '#666' }}>
          Select or drop a SQLite database file to explore its forecast data.
        </p>
        <input
          ref={inputRef}
          type="file"
          accept=".sqlite,.sqlite3,.db,.bin,application/octet-stream"
          style={{ display: 'none' }}
          onChange={(e) => handleFiles(e.target.files)}
        />
        <button
          onClick={() => inputRef.current?.click()}
          disabled={loading}
          style={{
            padding: '10px 20px',
            fontSize: 15,
            border: 'none',
            borderRadius: 6,
            background: '#2196f3',
            color: 'white',
            cursor: loading ? 'not-allowed' : 'pointer',
            opacity: loading ? 0.6 : 1
          }}
        >
          {loading ? 'Loading database...' : 'Select database file'}
        </button>
        {error && (
          <p style={{ marginTop: 16, color: '#d32f2f', fontSize: 14 }}>{error}</p>
        )}
        {onCancel && (
          <button
            onClick={onCancel}
            disabled={loading}
            style={{
              display: 'block',
              margin: '16px auto 0',
              padding: '6px 12px',
              fontSize: 13,
              border: '1px solid #c4c8cc',
              borderRadius: 6,
              background: 'white',
              cursor: loading ? 'not-allowed' : 'pointer'
            }}
          >
            Cancel
          </button>
        )}
      </div>
    </div>
  )
}
