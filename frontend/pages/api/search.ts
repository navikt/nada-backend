import type { NextApiRequest, NextApiResponse } from 'next'
import { components } from '../../lib/schema'

type SearchResultEntry = components['schemas']['SearchResultEntry']

const response: SearchResultEntry[] = [
  {
    url: 'banan',
    type: 'dataproduct',
    id: 'dp_1',
    name: 'DP 1',
    excerpt: 'Dette er en lang og fin beskrivelse av det første produktet, som stopper etter... ',
  },
  {
    url: 'kake',
    type: 'dataproduct',
    id: 'dp_2',
    name: 'DP 2',
    excerpt: 'Dette er en lang og fin beskrivelse av det andre produktet, som stopper etter... ',
  },
  {
    url: 'Shake',
    type: 'dataset',
    id: 'ds_1',
    name: 'DS 1',
    excerpt: 'Dette er en lang og fin beskrivelse av det første datasettet, som stopper etter... ',
  },
  {
    url: 'Shake',
    type: 'dataset',
    id: 'ds_2',
    name: 'DS 2',
    excerpt: 'Dette er en lang og fin beskrivelse av det første datasettet, som stopper etter... ',
  },
  {
    url: 'Cookie',
    type: 'datapackage',
    id: 'DPA_1',
    name: 'DPA 1',
    excerpt: 'Dette er en lang og fin beskrivelse av det første pakke, som stopper etter... ',
  },
  {
    url: 'Cookie',
    type: 'datapackage',
    id: 'DPA_2',
    name: 'DPA 2',
    excerpt: 'Dette er en lang og fin beskrivelse av den andre pakke, som stopper etter... ',
  },
]

export const handler = (req: NextApiRequest, res: NextApiResponse) => {
  const query = req.query["q"]
  let filteredResponse  = response.filter((r) => Object.values(r).join().toLowerCase().includes(query))

  res.status(200).json(filteredResponse)
}

export default handler
