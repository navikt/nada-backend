import SearchResult from './searchresult'
import styled from 'styled-components'
import { navGra60 } from '../../styles/constants'
import { SearchBoxProps } from '../search/search'
import { Loader, Panel } from '@navikt/ds-react'
import { SearchResultEntry } from '../../lib/schema_types'

const ResultsBox = styled.div`
  flex-grow: 1;
  padding: 15px;
`

export interface ResultProps {
  data: SearchResultEntry[]
  error: string
}

export function Results({ data, error }: ResultProps) {
  if (error) {
    return <div>error</div>
  }
  if (!data) {
    return (
      <div>
        {' '}
        <Loader transparent />
      </div>
    )
  }

  return (
    <ResultsBox>
      <Panel border role="navigation">
        {data.map((d) => {
          return <SearchResult key={d.id} searchResultEntry={d} />
        })}
      </Panel>
    </ResultsBox>
  )
}

export default Results
