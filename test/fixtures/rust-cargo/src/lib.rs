use std::collections::HashMap;
use crate::domain::model::Order;
use crate::infrastructure::repository::OrderRepository;
use self::submodule::Helper;
use super::parent_module::Something;
pub use crate::domain::Model;

pub mod submodule;
pub mod domain;
pub mod infrastructure;

pub fn process_order() {
    let order = Order::new(1);
    let repo = OrderRepository::new();
    repo.save(&order);
}
