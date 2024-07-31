use std::{
    ffi::CStr,
    os::raw::c_char,
    panic::{catch_unwind, AssertUnwindSafe},
};

pub(crate) fn c_char_to_str(c: *const c_char) -> &'static str {
    let cstr = unsafe { CStr::from_ptr(c) };
    cstr.to_str().unwrap()
}

pub(crate) fn c_char_to_vec(c: *const c_char) -> Vec<u8> {
    let cstr = unsafe { CStr::from_ptr(c) };
    cstr.to_bytes().to_vec()
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
