import { useRouter } from 'next/router'
import PageLayout from '../../components/pageLayout'
import useSWR, { SWRConfig } from 'swr'
import { Loader } from '@navikt/ds-react'
import styled from 'styled-components'
import { DataproductSchema } from '../../lib/schema_types'
import ReactMarkdown from 'react-markdown'
import { parseISO, format } from 'date-fns'

const fetcher = (url: string) => fetch(url).then((res) => res.json())
import { nb } from 'date-fns/locale'
import { GetServerSideProps } from 'next'
import DataProductSpinner from '../../components/lib/spinner'

export const getServerSideProps: GetServerSideProps = async (context) => {
  const id = context?.params?.id
  if (typeof id !== 'string') return { props: {} }
  const key = `http://localhost:3000/api/dataproducts/${id}`
  const dataproduct = await fetcher(key)

  return {
    props: {
      fallback: {
        key: dataproduct,
      },
    },
  }
}

interface DataProductProps {
  fallback?: DataproductSchema
}

interface DataProductDetailProps {
  id: string
}

const DataProductDetail = ({ id }: DataProductDetailProps) => {
  const { data, error } = useSWR<DataproductSchema>(
    `/api/dataproducts/${id}`,
    fetcher
  )

  if (error) return <div>Error</div>

  if (!data) return <DataProductSpinner />

  const humanizeDate = (isoDate: string) =>
    format(parseISO(isoDate), 'PPPP', { locale: nb })

  return (
    <div>
      <h1>{data.name}</h1>
      <p>
        Opprettet: {humanizeDate(data.created)} &ndash; Oppdatert:{' '}
        {humanizeDate(data.last_modified)}
      </p>
      <div>
        <ReactMarkdown>
          {data.description || '*ingen beskrivelse*'}
        </ReactMarkdown>
      </div>
    </div>
  )
}

const DataProduct = ({ fallback }: DataProductProps) => {
  const router = useRouter()
  const { id } = router.query

  if (typeof id !== 'string') return null

  return (
    <PageLayout>
      <SWRConfig value={{ fallback }}>
        <DataProductDetail id={id} />
      </SWRConfig>
    </PageLayout>
  )
}

export default DataProduct
