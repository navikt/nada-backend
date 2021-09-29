import { useRouter } from 'next/router'

const DataPackage = () => {
  const router = useRouter()
  const { id } = router.query

  return <p>DataPackage: {id}</p>
}

export default DataPackage
