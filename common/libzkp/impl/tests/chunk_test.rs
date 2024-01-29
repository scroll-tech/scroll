use log::log;
use prover::utils::get_block_trace_from_file;
use std::{ffi::CString, path::Path, thread::sleep, time::Duration};
use zkp::chunk;

#[test]
fn chunk_test() {
    let params = CString::new("/assets/test_params").expect("test_params conversion failed");
    let assets = CString::new("/assets/test_assets").expect("test_assets conversion failed");

    let trace_path = "/assets/traces/1_transfer.json".to_string();
    let chunk_trace = get_block_trace_from_file(Path::new(&trace_path));
    let json_str = serde_json::to_string(&chunk_trace).expect("Serialization failed");
    log::info!("json str {:?}", json_str);

    let c_string = CString::new(json_str).expect("CString conversion failed");
    let c_str_ptr = c_string.as_ptr();

    unsafe {
        chunk::init_chunk_prover(params.as_ptr(), assets.as_ptr());
        let mut count = 1;
        loop {
            count += 1;
            log::info!("count {:?}", count);
            chunk::gen_chunk_proof(c_str_ptr);
        }
    }
}
