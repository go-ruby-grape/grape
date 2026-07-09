# frozen_string_literal: true
#
# Usage of grape's three deterministic pieces from Ruby, under go-embedded-ruby
# (rbgo). Route matching, params coercion and response formatting need no live
# interpreter; running an endpoint body is the host's job.

require "grape"

# --- Router: (method, path) -> a match (route + path params, or 404/405). ---
router = Grape::Router.new
router.get("/users/:id")
router.post("/users")

m = router.match("GET", "/users/42")
puts "#{m.status} #{m.route.http_method} #{m.route.pattern} id=#{m.params["id"]}"  # ok GET /users/:id id=42
puts "missing path -> #{router.match("GET", "/nope").status}"                       # not_found
bad = router.match("DELETE", "/users/42")
puts "#{bad.status}, allowed: #{bad.allowed.join(", ")}"                            # method_not_allowed, allowed: GET

# --- Validator: `params do … end` coerces a raw Hash or raises. ---
validator = Grape::Validator.new do
  requires :id, type: Integer
  optional :status, type: String, values: ["draft", "live"], default: "draft"
end
p validator.validate({ "id" => "42" })                    # {"id" => 42, "status" => "draft"}
begin
  validator.validate({ "status" => "nope" })
rescue Grape::Exceptions::ValidationErrors => e
  puts "invalid: #{e.message}"                            # invalid: id is missing, status does not have a valid value
end

# --- Formatter: serialise a response value to json / txt / xml. ---
formatter = Grape::Formatter.new
body, mime = formatter.format("json", { "ok" => true })
puts "#{mime} #{body}"                                    # application/json {"ok":true}
puts Grape.default_status("POST")                         # 201
