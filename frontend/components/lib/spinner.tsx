import styled from 'styled-components'
import { Loader } from '@navikt/ds-react'

const CenteredSpinner = styled.div`
  margin: 10% auto;
`

export const DataProductSpinner = () => (
  <CenteredSpinner>
    <Loader size="2xlarge" transparent />
  </CenteredSpinner>
)

export default DataProductSpinner
