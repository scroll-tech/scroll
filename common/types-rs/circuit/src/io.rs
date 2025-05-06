#[allow(unused_imports, clippy::single_component_path_imports)]
use openvm::platform as openvm_platform;

/// Read the witnesses from the hint stream.
///
/// rkyv needs special alignment for its data structures, use a pre-aligned buffer with rkyv::access_unchecked
/// is more efficient than rkyv::access.
#[cfg(target_os = "zkvm")]
#[inline(always)]
pub fn read_witnesses_rkyv_raw() -> Vec<u8> {
    use std::alloc::{GlobalAlloc, Layout, System};
    openvm_rv32im_guest::hint_input();
    let mut len: u32 = 0;
    openvm_rv32im_guest::hint_store_u32!((&mut len) as *mut u32 as u32);
    let num_words = len.div_ceil(4);
    let size = (num_words * 4) as usize;
    let layout = Layout::from_size_align(size, 16).unwrap();
    let ptr_start = unsafe { System.alloc(layout) };
    let mut ptr = ptr_start;
    for _ in 0..num_words {
        openvm_rv32im_guest::hint_store_u32!(ptr as u32);
        ptr = unsafe { ptr.add(4) };
    }
    unsafe { Vec::from_raw_parts(ptr_start, len as usize, size) }
}

/// Read the witnesses from the hint stream.
pub fn read_witnesses() -> Vec<u8> {
    #[cfg(not(target_os = "zkvm"))]
    return openvm::io::read_vec(); // avoid compiler complaint
    #[cfg(target_os = "zkvm")]
    return read_witnesses_rkyv_raw();
}
