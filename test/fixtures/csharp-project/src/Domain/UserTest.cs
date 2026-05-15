using System;
using MyApp.Domain.Entities;
using Xunit;

namespace MyApp.Tests.Domain
{
    /// <summary>
    /// Test file - should be skipped by detector
    /// </summary>
    public class UserTests
    {
        [Fact]
        public void User_Creation_Valid()
        {
            var user = new User
            {
                Id = Guid.NewGuid(),
                Username = "testuser",
                Email = "test@example.com"
            };

            Assert.NotNull(user);
            Assert.Equal("testuser", user.Username);
        }
    }
}
