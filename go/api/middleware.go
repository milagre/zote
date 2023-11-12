package api

type Middleware func(req Request, next HandleFunc) ResponseBuilder

func NewCORSMiddleware() Middleware {
	return func(req Request, next HandleFunc) ResponseBuilder {
		resp := next(req)

		headers := resp.Headers()
		headers.Add("Access-Control-Allow-Origin", "*")
		headers.Add("Access-Control-Allow-Method", "*")
		headers.Add("Access-Control-Allow-Headers", "*")

		return BasicResponseReader(resp.Status(), headers, resp.Body())
	}
}
