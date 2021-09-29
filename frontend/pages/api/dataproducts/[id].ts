import type { NextApiRequest, NextApiResponse } from 'next'
import { Dataproduct } from '../../../lib/schema_types'

const response: Dataproduct = {
  id: 'test',
  name: 'test',
  description: `## Lorem ipsum
  Dolor *sit* **amet**`,
  slug: 'lorem ipsum',
  repo: 'lorem ipsum',
  last_modified: '2013-09-10T19:00Z',
  created: '1887-08-21T19:00Z',
  owner: {
    team: 'Test team',
    teamkatalogen: 'Test team',
  },
  keyword: ['lorem ipsum', 'sorryyy'],
  datasets: [
    {
      id: 'test dataset',
      type: 'bigquery',
    },
  ],
}

export const handler = (req: NextApiRequest, res: NextApiResponse) => {
  res.status(200).json(response)
}

export default handler
