import { Search } from 'lucide-react'
import styles from './SearchField.module.css'

interface SearchFieldProps {
  value: string
  onChange: (value: string) => void
  placeholder: string
  className?: string
}

export function SearchField({ value, onChange, placeholder, className }: SearchFieldProps) {
  return (
    <label className={[styles.search, className].filter(Boolean).join(' ')}>
      <Search size={13} />
      <input
        type="search"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  )
}
