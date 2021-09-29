import { useRouter } from 'next/router'

const Dataset = () => {
  const router = useRouter()
  const { id } = router.query

  return <p>Dataset: {id}</p>
}

export default Dataset
