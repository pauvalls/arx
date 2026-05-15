using System.Collections.Generic;
using System.Linq;
using MyApp.Domain.Entities;
using MyApp.Infrastructure.Persistence;

namespace MyApp.Infrastructure.Repositories
{
    /// <summary>
    /// Repository for User entity operations
    /// </summary>
    public class UserRepository
    {
        private readonly AppDbContext _context;

        public UserRepository(AppDbContext context)
        {
            _context = context;
        }

        public User GetById(Guid id)
        {
            return _context.Users.FirstOrDefault(u => u.Id == id);
        }

        public IEnumerable<User> GetAll()
        {
            return _context.Users.ToList();
        }

        public void Add(User user)
        {
            _context.Users.Add(user);
            _context.SaveChanges();
        }
    }
}
