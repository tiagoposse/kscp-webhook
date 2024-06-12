FROM golang as build

ENV CGO_ENABLED=0

ADD go.mod go.sum .
RUN go mod download
ADD main.go .
ADD internal internal

RUN go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /operator *.go


FROM alpine

COPY --from=build /operator /operator

ENTRYPOINT [ "/operator" ]