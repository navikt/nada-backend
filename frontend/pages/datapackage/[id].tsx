import { useRouter } from 'next/router'
import PageLayout from '../../components/pageLayout'
import useSWR from 'swr'
import fetcher from '../../lib/fetcher'

const DataPackage = () => {
  const router = useRouter()
  const { id } = router.query
  const { data, error } = useSWR('api/katalogen/datapackage', fetcher)

  return (
    <PageLayout>
      <p>DataPackage: {id}</p>
    </PageLayout>
  )
}

export default DataPackage
