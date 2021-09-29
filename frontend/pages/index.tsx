import type { NextPage } from "next"
import SearchBox from "../components/search/search"
import Results from "../components/results/results"
import PageLayout from "../components/pageLayout"
import { useEffect, useState } from "react"

const SearchPage: NextPage = () => {
  const [query, setQuery] = useState("")

  useEffect(() => {}, [query])
  console.log(query)
  return (
    <PageLayout>
      <SearchBox query={query} setQuery={setQuery} />
      <Results />
    </PageLayout>
  )
}

export default SearchPage
