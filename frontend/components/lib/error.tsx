interface errorMessageProps {
  error: Error
}

export const ErrorMessage = ({ error }: errorMessageProps) => {
  return <h1>{error.message}</h1>
}

export default ErrorMessage
