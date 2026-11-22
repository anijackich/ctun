export const errorMessage = (err: unknown) => String(err).replace(/^(Error: )?([a-z]+: )?/, '')
