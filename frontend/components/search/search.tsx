import styled from "styled-components"
import { Button, TextField } from "@navikt/ds-react"
import { Search } from "@navikt/ds-icons"
import React from "react"

const SearchDiv = styled.div`
  border: 2px solid black;
  width: 25%;
  padding: 15px;
  margin-right: 25px;
`
export interface SearchBoxProps {
  query: string
  setQuery: React.Dispatch<React.SetStateAction<string>>
}

export default function SearchBox({ query, setQuery }: SearchBoxProps) {
  const [searchBox, setSearchBox] = React.useState("")

  return (
    <SearchDiv>
      <form
        onSubmit={(e) => { e.preventDefault()
          setQuery(searchBox)
        }}
      >
        <TextField 
          onChange={(event) => setSearchBox(event.target.value)}
          label={""}
        />
        <Button>
          <Search onClick={() => setQuery(searchBox)} /> Søk
        </Button>
      </form>
    </SearchDiv>
  )
}
