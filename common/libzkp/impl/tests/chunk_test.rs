use log::info;
use prover::utils::get_block_trace_from_file;
use std::{
    ffi::{CStr, CString},
    path::Path,
};
use zkp::chunk;

#[test]
fn chunk_test() {
    println!("start chunk_test.");
    unsafe {
        let params = CString::new("/assets/test_params").expect("test_params conversion failed");
        let assets = CString::new("/assets/test_assets").expect("test_assets conversion failed");

        let trace_path = "/assets/traces/1_transfer.json".to_string();
        let chunk_trace = get_block_trace_from_file(Path::new(&trace_path));
        let json_str = serde_json::to_string(&chunk_trace).expect("Serialization failed");
        println!("json str: {}", json_str);

        let c_string = CString::new(json_str).expect("CString conversion failed");
        let c_str_ptr = c_string.as_ptr();

        chunk::init_chunk_prover(params.as_ptr(), assets.as_ptr());
        let mut count = 1;
        // loop {
        count += 1;
        println!("count {:?}", count);

        let ret = chunk::gen_chunk_proof(c_str_ptr);
        let ret_cstr = CStr::from_ptr(ret)
            .to_str()
            .expect("Failed to convert C string to Rust string");
        println!("ret: {:?}", ret_cstr)
        // }
    }
}
