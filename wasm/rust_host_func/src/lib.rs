use wasmedge_bindgen::*;
use wasmedge_bindgen_macro::*;

extern "C" {
    fn fetch(url_pointer: *const u8, url_length: i32) -> i32;
    fn http(request_json_pointer: *const u8, request_json_length: i32) -> i32;
    fn write_mem(pointer: *const u8);
}

// Define return structure
#[derive(Debug)]
pub struct ProcessResult {
    pub processed_u8: u8,
    pub bytes_len: i32,
    pub appended_string: String,
    pub vector_sum: i32,
    pub reversed_string: String,
    pub doubled_vector: Vec<i32>,
}

#[wasmedge_bindgen]
pub unsafe extern "C" fn call_google() -> i32 {
    let url = "https://www.google.com";
    let pointer = url.as_bytes().as_ptr();

    // call host function to fetch the source code, return the result length
    let res_len = fetch(pointer, url.len() as i32) as usize;

    // malloc memory
    let mut buffer = Vec::with_capacity(res_len);
    let pointer = buffer.as_mut_ptr();

    // call host function to write source code to the memory
    write_mem(pointer);

    // find occurrences from source code
    buffer.set_len(res_len);
    let str = std::str::from_utf8(&buffer).unwrap();
    str.matches("google").count() as i32
}

#[wasmedge_bindgen]
pub unsafe extern "C" fn say(s: String) -> String {
    let r = String::from("hello ");
    return r + s.as_str();
}

// HTTP functionality test function - GET request
#[wasmedge_bindgen]
pub unsafe extern "C" fn test_http_get() -> String {
    // Build HTTP GET request JSON
    let http_request = r#"{
        "method": "GET",
        "url": "https://httpbin.org/get",
        "headers": {
            "User-Agent": "WasmVM-TEE/1.0"
        },
        "timeout": 30
    }"#;

    let request_ptr = http_request.as_bytes().as_ptr();
    let request_len = http_request.len() as i32;

    // Call host's http function
    let response_len = http(request_ptr, request_len) as usize;

    // Allocate memory to receive response
    let mut response_buffer = Vec::with_capacity(response_len);
    let buffer_ptr = response_buffer.as_mut_ptr();

    // Call host function to write response to memory
    write_mem(buffer_ptr);

    // Set buffer length and convert to string
    response_buffer.set_len(response_len);
    match std::str::from_utf8(&response_buffer) {
        Ok(response_str) => response_str.to_string(),
        Err(_) => "Failed to parse response".to_string(),
    }
}

// HTTP functionality test function - POST request
#[wasmedge_bindgen]
pub unsafe extern "C" fn test_http_post() -> String {
    // Build HTTP POST request JSON
    let http_request = r#"{
        "method": "POST",
        "url": "https://httpbin.org/post",
        "headers": {
            "Content-Type": "application/json",
            "User-Agent": "WasmVM-TEE/1.0"
        },
        "body": "{\"message\": \"Hello from WASM!\", \"timestamp\": 1234567890}",
        "timeout": 30
    }"#;

    let request_ptr = http_request.as_bytes().as_ptr();
    let request_len = http_request.len() as i32;

    // Call host's http function
    let response_len = http(request_ptr, request_len) as usize;

    // Allocate memory to receive response
    let mut response_buffer = Vec::with_capacity(response_len);
    let buffer_ptr = response_buffer.as_mut_ptr();

    // Call host function to write response to memory
    write_mem(buffer_ptr);

    // Set buffer length and convert to string
    response_buffer.set_len(response_len);
    match std::str::from_utf8(&response_buffer) {
        Ok(response_str) => response_str.to_string(),
        Err(_) => "Failed to parse response".to_string(),
    }
}

// HTTP functionality test function - request with custom headers
#[wasmedge_bindgen]
pub unsafe extern "C" fn test_http_with_headers() -> String {
    // Build HTTP request with multiple custom headers
    let http_request = r#"{
        "method": "GET",
        "url": "https://httpbin.org/headers",
        "headers": {
            "X-Custom-Header": "test-value",
            "Authorization": "Bearer fake-token",
            "Accept": "application/json",
            "User-Agent": "WasmVM-TEE/1.0"
        },
        "timeout": 30
    }"#;

    let request_ptr = http_request.as_bytes().as_ptr();
    let request_len = http_request.len() as i32;

    // Call host's http function
    let response_len = http(request_ptr, request_len) as usize;

    // Allocate memory to receive response
    let mut response_buffer = Vec::with_capacity(response_len);
    let buffer_ptr = response_buffer.as_mut_ptr();

    // Call host function to write response to memory
    write_mem(buffer_ptr);

    // Set buffer length and convert to string
    response_buffer.set_len(response_len);
    match std::str::from_utf8(&response_buffer) {
        Ok(response_str) => response_str.to_string(),
        Err(_) => "Failed to parse response".to_string(),
    }
}

// Return complex results using JSON string
#[wasmedge_bindgen]
pub unsafe extern "C" fn process_complex_types_json(
    input_u8: u8,
    input_bytes: Vec<u8>,
    input_string: String,
    input_vector: Vec<i32>,
) -> String {
    let processed_u8 = input_u8.wrapping_add(10);
    let processed_bytes_len = input_bytes.len();
    let appended_string = format!("{} processed", input_string);
    let sum_of_vector: i32 = input_vector.iter().sum();
    let original_string_reversed: String = input_string.chars().rev().collect();
    let returned_vector: Vec<i32> = input_vector.iter().map(|&x| x * 2).collect();

    // Return JSON formatted string
    format!(
        r#"{{"processed_u8": {}, "bytes_len": {}, "appended_string": "{}", "vector_sum": {}, "reversed_string": "{}", "doubled_vector": {:?}}}"#,
        processed_u8,
        processed_bytes_len,
        appended_string,
        sum_of_vector,
        original_string_reversed,
        returned_vector
    )
}
