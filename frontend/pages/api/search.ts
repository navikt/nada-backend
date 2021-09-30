import type { NextApiRequest, NextApiResponse } from 'next'
import { components } from '../../lib/schema'

type SearchResultEntry = components['schemas']['SearchResultEntry']

const response: SearchResultEntry[] = [
  {
    url: 'banan',
    type: 'dataproduct',
    id: '1',
    name: 'Yohhoo',
    excerpt: 'Første ',
  },
  {
    url: 'kake',
    type: 'dataproduct',
    id: '2',
    name: 'Andre',
    excerpt: 'Andre ',
  },

  {
    url: 'Shake',
    type: 'dataset',
    id: '3',
    name: 'Strawberry',
    excerpt: 'Strawberry ',
  },

  {
    url: 'Cookie',
    type: 'datapackage',
    id: '4',
    name: 'Oaths',
    excerpt: 'Oaths ',
  },
]

export const handler = (req: NextApiRequest, res: NextApiResponse) => {
  res.status(200).json(response)
}

export default handler
