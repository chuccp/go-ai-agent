import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { usePackageStore } from '@/stores/packageStore'

export default function PackageManager() {
  const { t } = useTranslation()
  const { packages, loading, fetchPackages, importPackage, exportPackage, deletePackage } = usePackageStore()

  useEffect(() => { fetchPackages() }, [fetchPackages])

  const handleImport = () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.zip'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (file) await importPackage(file)
    }
    input.click()
  }

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '8px 10px', border: '1px solid #d0d5dd', borderRadius: 8, fontSize: 13, outline: 'none',
  }

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', background: '#f2f4f7' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px', background: '#fff', borderBottom: '0.5px solid rgba(16,24,40,0.08)' }}>
        <a href="#/" style={{ display: 'flex', alignItems: 'center', color: '#676f83', textDecoration: 'none', fontSize: 13, gap: 4 }}>
          ← {t('common.back')}
        </a>
        <div style={{ flex: 1 }} />
        <button onClick={handleImport} style={{ padding: '7px 16px', borderRadius: 8, border: 'none', background: '#155aef', color: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer' }}>
          {t('common.import')}
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 20 }}>
        {loading && <div style={{ color: '#676f83' }}>{t('common.loading')}</div>}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 14 }}>
          {packages.map(p => (
            <div key={p.id} style={{ background: '#fff', borderRadius: 12, border: '0.5px solid rgba(16,24,40,0.08)', padding: 16, display: 'flex', flexDirection: 'column', gap: 8 }}>
              <div style={{ fontSize: 15, fontWeight: 600, color: '#101828' }}>{p.icon || '📦'} {p.name}</div>
              <div style={{ fontSize: 12, color: '#676f83' }}>{p.package_id} · v{p.version} · {p.kind}</div>
              <div style={{ fontSize: 12, color: '#98a2b3' }}>{p.description}</div>
              <div style={{ display: 'flex', gap: 8, marginTop: 'auto' }}>
                <button onClick={() => exportPackage(p.id, p.name)} style={{ ...inputStyle, width: 'auto', padding: '6px 14px', cursor: 'pointer' }}>{t('common.export')}</button>
                <button onClick={() => { if (window.confirm(t('package.confirmDelete') || 'Delete?')) deletePackage(p.id) }} style={{ ...inputStyle, width: 'auto', padding: '6px 14px', color: '#f04438', borderColor: '#f0443833', cursor: 'pointer' }}>{t('common.delete')}</button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
