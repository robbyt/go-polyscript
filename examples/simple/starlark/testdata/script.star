# Script has access to ctx variable passed from Go
name = ctx["name"]
message = "Hello, " + name + "!"

# Return a map with our result
result = {
    "greeting": message,
    "length": len(message)
}