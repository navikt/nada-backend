import { useRouter } from 'next/router'

const DataProduct = () => {
  const router = useRouter()
  const { id } = router.query

  return <p>DataProduct: {id}</p>
}

export default DataProduct
