use once_cell::sync::Lazy;
use std::{
    env,
    ffi::{CStr, CString},
    os::raw::c_char,
    panic::{catch_unwind, AssertUnwindSafe},
    path::PathBuf,
};

// Only used for debugging.
pub(crate) static OUTPUT_DIR: Lazy<Option<String>> =
    Lazy::new(|| env::var("PROVER_OUTPUT_DIR").ok());

/// # Safety
#[no_mangle]
pub extern "C" fn free_c_chars(ptr: *mut c_char) {
    if ptr.is_null() {
        log::warn!("Try to free an empty pointer!");
        return;
    }

    unsafe {
        let _ = CString::from_raw(ptr);
    }
}

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

pub(crate) fn vec_to_c_char(bytes: Vec<u8>) -> *mut c_char {
    CString::new(bytes).unwrap().into_raw()
}

pub(crate) fn file_exists(dir: &str, filename: &str) -> bool {
    let mut path = PathBuf::from(dir);
    path.push(filename);

    path.exists()
}

pub(crate) fn panic_catch<F: FnOnce() -> R, R>(f: F) -> Result<R, String> {
    catch_unwind(AssertUnwindSafe(f)).map_err(|err| {
        if let Some(s) = err.downcast_ref::<String>() {
            s.to_string()
        } else if let Some(s) = err.downcast_ref::<&str>() {
            s.to_string()
        } else {
            format!("unable to get panic info {err:?}")
        }
    })
}
