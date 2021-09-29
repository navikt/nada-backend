import type { NextApiRequest, NextApiResponse } from 'next'
import { components } from '../../lib/schema'

type SearchResultEntry = components['schemas']['SearchResultEntry']

const response: SearchResultEntry[] = [
  {
    url: 'banan',
    type: 'dataproduct',
    id: 'asdad',
    name: 'Yohhoo',
    excerpt: 'Første ',
  },
  {
    url: 'kake',
    type: 'dataproduct',
    id: 'asdad',
    name: 'Andre',
    excerpt: 'Andre ',
  },

  {
    url: 'Shake',
    type: 'dataset',
    id: 'shake',
    name: 'Strawberry',
    excerpt: 'Strawberry ',
  },

  {
    url: 'Cookie',
    type: 'datapackage',
    id: 'fakeCookie',
    name: 'Oaths',
    excerpt: 'Oaths ',
  },
]

export const handler = (req: NextApiRequest, res: NextApiResponse) => {
  res.status(200).json(response)
}

export default handler
