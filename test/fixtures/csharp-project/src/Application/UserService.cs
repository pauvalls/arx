using System;
using MyApp.Domain.Entities;
using MyApp.Infrastructure.Repositories;

namespace MyApp.Application.Services
{
    /// <summary>
    /// Application service for user operations
    /// </summary>
    public class UserService
    {
        private readonly UserRepository _repository;

        public UserService(UserRepository repository)
        {
            _repository = repository;
        }

        public User GetUserById(Guid id)
        {
            return _repository.GetById(id);
        }

        public void RegisterUser(string username, string email, string password)
        {
            var user = new User
            {
                Id = Guid.NewGuid(),
                Username = username,
                Email = email,
                PasswordHash = password // In real app, hash this!
            };
            _repository.Add(user);
        }
    }
}
