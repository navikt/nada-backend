export const fetcher = async (url: string) =>
  fetch(url).then((res) => {
    if (!res.ok) {
      const error = new Error(`${res.status} - ${res.statusText}`)

      throw error
    }
    return res.json()
  })
export default fetcher
