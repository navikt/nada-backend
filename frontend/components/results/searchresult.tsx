import styled from 'styled-components'
import Link from 'next/link'
import { navGra20, navGra40 } from '../../styles/constants'
import { components } from '../../lib/schema'
import BigQueryLogo from '../lib/icons/bigQueryLogo'
import DataPackageLogo from '../lib/icons/dataPackageLogo'
import DataProductLogo from '../lib/icons/dataProductLogo'


const SearchResultDiv = styled.div`
  background-color: ${navGra20};
  padding: 5px;
  margin-bottom: 5px;
  h1 {
    font-size: 1.5em;
    margin: 0;
  }
  cursor: pointer;
  :hover {
    background-color: ${navGra40};
  }
`

const SearchResultLogo = styled.div`
  display: flex;
  justify-content: flex-end;
  flex: 1;
  align-items: center;
`

const SearchResultContent = styled.div`
  display: flex;
  flex: 1;
  padding: 8px;
`

type SearchResultEntry = components['schemas']['SearchResultEntry']

export interface SearchResultProps {
  searchResultEntry: SearchResultEntry
}

export const SearchResult = ({ searchResultEntry }: SearchResultProps) => {
  if (searchResultEntry.type === 'dataproduct') {
    return (
      <SearchResultDiv>
        <Link href={`/dataproduct/${searchResultEntry.id}`}>
          <SearchResultContent>
            <div>
              <h1>{searchResultEntry.name}</h1>
              <p>{searchResultEntry.excerpt}</p>
            </div>
            <SearchResultLogo><DataProductLogo/></SearchResultLogo>
          </SearchResultContent>
        </Link>
      </SearchResultDiv>
    )
  } else if (searchResultEntry.type === 'datapackage') {
    return (
      <SearchResultDiv>
        <Link href={`/datapackage/${searchResultEntry.id}`}>
          <SearchResultContent>
            <div>
              <h1>{searchResultEntry.name}</h1>
              <p>{searchResultEntry.excerpt}</p>
            </div>
            <SearchResultLogo><DataPackageLogo /></SearchResultLogo>
          </SearchResultContent>
        </Link>
      </SearchResultDiv>
    )
  }

  return (
    <SearchResultDiv>
      <Link href={`/dataset/${searchResultEntry.id}`}>
        <SearchResultContent>
          <div>
            <h1>{searchResultEntry.name}</h1>
            <p>{searchResultEntry.excerpt}</p>
          </div>
          <SearchResultLogo><BigQueryLogo /></SearchResultLogo>
        </SearchResultContent>
      </Link>
    </SearchResultDiv>
  )
}

export default SearchResult
