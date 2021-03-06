# Balanced Leaky Queue

`blqueue` balances queue among available workers (`WorkersMin` and `WorkersMax` params).

It has the following features:
* Put to sleep idle workers depending of queue's fullness rate.
* Resume sleeping workers when fullness rate of the queue grows over the limit.
* Use [leaky bucket](https://en.wikipedia.org/wiki/Leaky_bucket) algorithm on newly comes elements when queue is full.
* Leaked elements may be "catch" by special helper to perform some actions on them. 

## See

* https://en.wikipedia.org/wiki/Leaky_bucket
* https://golang.org/doc/effective_go#leaky_buffer
* https://en.wikipedia.org/wiki/Thread_pool
