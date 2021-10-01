import type { NextApiRequest, NextApiResponse } from 'next'
import { DatasetSchema } from '../../../lib/schema_types'

const response: DatasetSchema = {
  id: 'DS_1',
  dataproduct_id: 'DP_1',
  name: 'DS 1',
  description: `## Lorem ipsum
  Dolor *sit* **amet**`,
  pii: false,
  bigquery: {
    project_id: 'projectid',
    dataset: 'datasetid',
    table: 'tableid',
  },
}

export const handler = (req: NextApiRequest, res: NextApiResponse) => {
  res.status(200).json(response)
}

export default handler
