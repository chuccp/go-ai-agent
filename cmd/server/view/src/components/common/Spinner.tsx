export default function Spinner({ size = 24 }: { size?: number }) {
  return (
    <div
      style={{
        width: size, height: size, border: '2px solid #e2e8f0',
        borderTopColor: '#155aef', borderRadius: '50%',
        animation: 'spin 0.6s linear infinite',
      }}
    />
  )
}
