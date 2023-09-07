use once_cell::sync::Lazy;
use std::{
    env,
    ffi::{CStr, CString},
    os::raw::c_char,
    path::PathBuf,
};

// Only used for debugging.
pub(crate) static OUTPUT_DIR: Lazy<Option<String>> =
    Lazy::new(|| env::var("PROVER_OUTPUT_DIR").ok());

pub(crate) fn c_char_to_str(c: *const c_char) -> &'static str {
    let cstr = unsafe { CStr::from_ptr(c) };
    cstr.to_str().unwrap()
}

pub(crate) fn c_char_to_vec(c: *const c_char) -> Vec<u8> {
    let cstr = unsafe { CStr::from_ptr(c) };
    cstr.to_bytes().to_vec()
}

pub(crate) fn string_to_c_char(string: String) -> *const c_char {
    CString::new(string).unwrap().into_raw()
}

pub(crate) fn vec_to_c_char(bytes: Vec<u8>) -> *const c_char {
    CString::new(bytes).unwrap().into_raw()
}

pub(crate) fn file_exists(dir: &str, filename: &str) -> bool {
    let mut path = PathBuf::from(dir);
    path.push(filename);

    path.exists()
}
