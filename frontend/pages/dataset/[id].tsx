import {useRouter} from 'next/router'
import useSWR from "swr";
import DataProductSpinner from "../../components/lib/spinner";
import PageLayout from "../../components/pageLayout";
import {DatasetSchema} from "../../lib/schema_types";
import ReactMarkdown from "react-markdown";
import ErrorMessage from "../../components/lib/error";

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface DatasetDetailProps {
    data: DatasetSchema
    error: Error
}

const DatasetDetail = ({data, error}: DatasetDetailProps) => {

    if (error) return <ErrorMessage error={error}/>

    if (!data) return <DataProductSpinner/>

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
    const {id} = router.query

    const {data, error} = useSWR(id ? `/api/dataset/${id}` : null, fetcher)
    if (error) {console.log(error.toString());}

    return (
        <PageLayout>
            <DatasetDetail data={data} error={error}/>
        </PageLayout>
    )

}

export default Dataset
