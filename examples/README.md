# Ruby examples

Pure-Ruby examples for `grape`'s deterministic pieces (routing, params coercion,
response formatting), provided by
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) (rbgo). Run them
with the `rbgo` interpreter:

```sh
rbgo examples/grape_usage.rb
```

| File | Shows |
| --- | --- |
| [`grape_usage.rb`](grape_usage.rb) | `Grape::Router` matching (404/405), params coercion, response formatting. |

Each example is executed as-is under rbgo (`require "grape"`).
