use aggregator::init_zstd_encoder;
use core::slice;
use std::io::Write;
use std::os::raw::{c_char, c_uchar};
use std::ptr::null;

fn out_as_err(err: &str, out: &mut [u8]){
    assert!(out.len() > 8, "the buffer is too few to containt any msg");
    let msg = if err.len() > out.len() + 1{
        "NENOUGH"
    } else {
        err
    };

    let cpy_src = unsafe { slice::from_raw_parts(msg.as_ptr(), msg.len()) };
    out[..cpy_src.len()].copy_from_slice(cpy_src);
    out[cpy_src.len()] = 0; // build the c-stype string
}

/// Entry
#[no_mangle]
pub unsafe extern "C" fn compress_scroll_batch_bytes(
    src: *const c_uchar,
    src_size: u64,
    output_buf: *mut c_uchar,
    output_buf_size: *mut u64,
) -> *const c_char {

    let buf_size = *output_buf_size;
    let src = unsafe { slice::from_raw_parts(src, src_size as usize) };
    let out = unsafe { slice::from_raw_parts_mut(output_buf, buf_size as usize)};

    let mut encoder = init_zstd_encoder();
    encoder
        .set_pledged_src_size(Some(src.len() as u64))
        .expect("infallible");

    let ret = encoder.write_all(src);
    let ret = ret.and_then(|_|encoder.finish());
    if let Err(e) = ret {
        out_as_err(e.to_string().as_str(), out);
        return output_buf as *const c_char
    }

    let ret = ret.unwrap();
    out[..ret.len()].copy_from_slice(ret.as_slice());
    *output_buf_size = ret.len() as u64;

    null()
}
