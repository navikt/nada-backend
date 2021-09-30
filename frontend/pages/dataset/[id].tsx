import { useRouter } from 'next/router'
import useSWR from "swr";
import DataProductSpinner from "../../components/lib/spinner";
import PageLayout from "../../components/pageLayout";
import {DatasetSchema} from "../../lib/schema_types";
import ReactMarkdown from "react-markdown";

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface DatasetDetailProps {
  data: DatasetSchema
}

const DatasetDetail = ({data}: DatasetDetailProps) => {

  if (!data) return <DataProductSpinner />

  return (
    <div>
      <h1>{data.name}</h1>

      <div>
        <ReactMarkdown>
          {data.description || '*ingen beskrivelse*'}
        </ReactMarkdown>
      </div>
      <h3>{data.pii}</h3>
      <p>
        {data.bigquery?.project_id + '-' + data.bigquery?.dataset + '-' + data.bigquery?.table}
      </p>

    </div>
  )
}

const Dataset = () => {
  const router = useRouter()
  const { id } = router.query

  const { data } = useSWR(id ?  `/api/dataset/${id}` : null, fetcher)

  return (
      <PageLayout>
        <DatasetDetail data={data}/>
      </PageLayout>
  )

}

export default Dataset
