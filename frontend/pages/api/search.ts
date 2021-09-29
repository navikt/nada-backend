import type { NextApiRequest, NextApiResponse } from "next"

export const handler = (req: NextApiRequest, res: NextApiResponse) => {
  res.status(200).json({ text: "Hello" })
}

export default handler