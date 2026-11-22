import { CircleAlert, LoaderCircle, X } from 'lucide-react'
import { useEffect } from 'react'
import { useT } from '../i18n'
import { useApp } from '../store/app'
import styles from './Notice.module.css'

const AUTO_DISMISS_MS = 8000

export function Notice() {
  const t = useT()
  const notice = useApp((s) => s.notice)
  const importing = useApp((s) => s.importing)
  const setNotice = useApp((s) => s.setNotice)

  useEffect(() => {
    if (!notice) return
    const timer = setTimeout(() => setNotice(undefined), AUTO_DISMISS_MS)
    return () => clearTimeout(timer)
  }, [notice, setNotice])

  if (!notice && !importing) return null
  return (
    <div className={styles.notice} role={notice ? 'alert' : 'status'} data-error={notice ? '' : undefined}>
      {notice ? (
        <>
          <CircleAlert size={15} className={styles.icon} />
          <span className={styles.text}>{notice}</span>
          <button className={styles.close} onClick={() => setNotice(undefined)} aria-label={t.notice.dismiss}>
            <X size={13} strokeWidth={2.4} />
          </button>
        </>
      ) : (
        <>
          <LoaderCircle size={15} className={styles.spin} />
          <span className={styles.text}>{t.common.checking}</span>
        </>
      )}
    </div>
  )
}
