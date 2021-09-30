import type { NextApiRequest, NextApiResponse } from 'next'
import { DatasetSchema } from '../../../lib/schema_types'

const response: DatasetSchema = {
  id: 'test',
  dataproduct_id: '',
  name: 'test',
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
