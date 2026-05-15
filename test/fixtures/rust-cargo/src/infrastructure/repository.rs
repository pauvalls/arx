use crate::domain::model::Order;

pub struct OrderRepository;

impl OrderRepository {
    pub fn new() -> Self {
        Self
    }

    pub fn save(&self, _order: &Order) {
        // save to database
    }
}
