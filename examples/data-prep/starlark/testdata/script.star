# Script has access to ctx variable passed from Go
def process_data():
    # Access data with fallbacks for compatibility with both patterns
    input_data = ctx.get("input_data", {})
    
    # Access static config data (set at compile time)
    app_version = ctx.get("app_version", "unknown")
    environment = ctx.get("environment", "unknown")
    config = ctx.get("config", {})
    
    # Get name from runtime data
    name = input_data.get("name", "Default")
    
    # Get timestamp from runtime data
    timestamp = input_data.get("timestamp", "Unknown")
    
    # Process user data from runtime data
    user_data = input_data.get("user_data", {})
    user_role = user_data.get("role", "guest")
    user_id = user_data.get("id", "unknown")
    
    # Access request data if available
    request = input_data.get("request", {})
    request_method = request.get("Method", "")
    request_path = request.get("URL_Path", "")
    
    # Construct result dictionary
    result = {
        "greeting": "Hello, " + name + "!",
        "timestamp": timestamp,
        "message": "Processed by " + user_role + " at " + timestamp,
        "user_id": user_id,
        "request_info": request_method + " " + request_path,
        "app_info": {
            "version": app_version,
            "environment": environment,
            "features": config.get("feature_flags", {})
        }
    }
    
    return result

# Call the function and store the result
result = process_data()
# The underscore variable is what gets returned to the VM
_ = result