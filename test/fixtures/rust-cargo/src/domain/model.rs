pub struct Order {
    id: u64,
}

impl Order {
    pub fn new(id: u64) -> Self {
        Self { id }
    }
}
