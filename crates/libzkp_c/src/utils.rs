use std::{ffi::CStr, os::raw::c_char};

pub(crate) fn c_char_to_str(c: *const c_char) -> &'static str {
    let cstr = unsafe { CStr::from_ptr(c) };
    cstr.to_str().unwrap()
}

pub(crate) fn c_char_to_vec(c: *const c_char) -> Vec<u8> {
    let cstr = unsafe { CStr::from_ptr(c) };
    cstr.to_bytes().to_vec()
}
