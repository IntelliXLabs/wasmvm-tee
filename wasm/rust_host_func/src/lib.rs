use wasmedge_bindgen::*;
use wasmedge_bindgen_macro::*;

extern "C" {
    fn fetch(url_pointer: *const u8, url_length: i32) -> i32;
    fn write_mem(pointer: *const u8);
}

// 定义返回结构体
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

// 使用 JSON 字符串返回复杂结果
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

    // 返回 JSON 格式的字符串
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
