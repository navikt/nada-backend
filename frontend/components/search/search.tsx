import styled from 'styled-components'
import {Button, SearchField, SearchFieldInput, TextField} from '@navikt/ds-react'
import {Close, Search} from '@navikt/ds-icons'
import React from 'react'
import {SearchFieldButton, SearchFieldClearButton} from "@navikt/ds-react/esm/form/search-field";

const SearchDiv = styled.div`
  width: 80%;
  padding: 15px;
  margin: 0 auto;
`

export interface SearchBoxProps {
    query: string
    setQuery: React.Dispatch<React.SetStateAction<string>>
}

export default function SearchBox({query, setQuery}: SearchBoxProps) {
    const [searchBox, setSearchBox] = React.useState('')

    return (
        <SearchDiv role="navigation">
            <form
                onSubmit={(e) => {
                    e.preventDefault()
                    setQuery(searchBox)
                }}
            >
                <div style={{display: "flex", flex: 1, width: "100%"}}>
                    <SearchField label={""} description={""}>
                        <SearchFieldInput
                            onChange={(event) => setSearchBox(event.target.value)}
                        />
                        <SearchFieldClearButton>
                            <Close/>
                        </SearchFieldClearButton>
                        <SearchFieldButton onClick={() => setQuery(searchBox)}>
                            <Search/> Søk
                        </SearchFieldButton>
                    </SearchField>
                </div>
            </form>
        </SearchDiv>
    )
}
