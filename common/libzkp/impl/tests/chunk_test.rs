use glob::glob;
use prover::{utils::get_block_trace_from_file, BlockTrace};
use std::ffi::{CStr, CString};
use zkp::chunk;

#[test]
fn chunk_test() {
    println!("start chunk_test.");
    unsafe {
        let params = CString::new("/assets/test_params").expect("test_params conversion failed");
        let assets = CString::new("/assets/test_assets").expect("test_assets conversion failed");

        chunk::init_chunk_prover(params.as_ptr(), assets.as_ptr());

        let chunk_trace = load_batch_traces().1;
        let json_str = serde_json::to_string(&chunk_trace).expect("Serialization failed");
        let c_string = CString::new(json_str).expect("CString conversion failed");
        let c_str_ptr = c_string.as_ptr();

        let ptr_cstr = CStr::from_ptr(c_str_ptr)
            .to_str()
            .expect("Failed to convert C string to Rust string");

        println!("c_str_ptr len: {:?}", ptr_cstr.len());

        let mut count = 1;
        loop {
            count += 1;
            println!("count {:?}", count);

            let _ = chunk::gen_chunk_proof(c_str_ptr);
            // let ret_cstr = CStr::from_ptr(ret)
            //     .to_str()
            //     .expect("Failed to convert C string to Rust string");
            // println!("ret: {:?}", ret_cstr)
        }
    }
}

fn load_batch_traces() -> (Vec<String>, Vec<BlockTrace>) {
    let file_names: Vec<String> = glob(&"/assets/traces/1_transfer.json".to_string())
        .unwrap()
        .map(|p| p.unwrap().to_str().unwrap().to_string())
        .collect();
    log::info!("test batch with {:?}", file_names);
    let mut names_and_traces = file_names
        .into_iter()
        .map(|trace_path| {
            let trace: BlockTrace = get_block_trace_from_file(trace_path.clone());
            (
                trace_path,
                trace.clone(),
                trace.header.number.unwrap().as_u64(),
            )
        })
        .collect::<Vec<_>>();
    names_and_traces.sort_by(|a, b| a.2.cmp(&b.2));
    log::info!(
        "sorted: {:?}",
        names_and_traces
            .iter()
            .map(|(f, _, _)| f.clone())
            .collect::<Vec<String>>()
    );
    names_and_traces.into_iter().map(|(f, t, _)| (f, t)).unzip()
}
