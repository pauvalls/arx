using System;

namespace MyApp.Domain.Entities
{
    /// <summary>
    /// User entity representing a user in the system
    /// </summary>
    public class User : BaseEntity
    {
        public string Username { get; set; }
        public string Email { get; set; }
        public string PasswordHash { get; set; }
    }
}
