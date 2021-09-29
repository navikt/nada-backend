import type { NextApiRequest, NextApiResponse } from 'next'
import { Dataproduct } from '../../../lib/schema_types'

const response: Dataproduct = {
  id: 'test',
  name: 'test',
  description: 'lorem ipsum',
  slug: 'lorem ipsum',
  repo: 'lorem ipsum',
  last_modified: 'lorem ipsum',
  created: 'lorem ipsum',
  owner: {
    team: 'Test team',
    teamkatalogen: 'Test team',
  },
  keyword: ['lorem ipsum', 'sorryyy'],
  datasets: [{
    id: 'test dataset',
    type: 'bigquery'
  }],
}

export const handler = (req: NextApiRequest, res: NextApiResponse) => {
  res.status(200).json(response)
}

export default handler