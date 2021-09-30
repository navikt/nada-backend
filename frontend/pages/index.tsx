import type { NextPage } from 'next'
import SearchBox from '../components/search/search'
import Results from '../components/results/results'
import PageLayout from '../components/pageLayout'
import { useEffect, useState } from 'react'
import useSWR from 'swr'

const fetcher = (url: string) => fetch(url).then((res) => res.json())

const SearchPage: NextPage = () => {
  const [query, setQuery] = useState('')
  const { data, error } = useSWR(`api/search?q=${query}`, fetcher)

  return (
    <PageLayout>
      <SearchBox query={query} setQuery={setQuery} />
      <Results data={data} error={error} />
    </PageLayout>
  )
}

export default SearchPage
