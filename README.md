# Numbers

## API

### Merge numbers

Takes URLs by query parameter `u`, creates external requests and merge results.
Result array will be sorted set. External requests limited by the time: 500 ms.

`> GET /numbers?u=http://example.com/primes&u=http://foobar.com/fibo`

Response:

`Status: 200 OK`
``{ "numbers": [ 1, 2, 3, 5, 8, 13 ] }`

## Start

`make run`

## Tests

`make test`
