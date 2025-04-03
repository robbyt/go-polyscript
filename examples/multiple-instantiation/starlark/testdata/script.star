# Script has access to ctx variable passed from Go
name = ctx["name"]
message = "Hello, " + name + "!"

# Return a dictionary with our result
result = {
    "greeting": message,
    "length": len(message)
}
# The last expression's value becomes the script's return value
result