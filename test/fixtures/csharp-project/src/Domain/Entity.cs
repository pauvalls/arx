using System;
using MyApp.Domain.Entities;
using MyApp.Domain.ValueObjects;

namespace MyApp.Domain
{
    /// <summary>
    /// Base entity class for domain models
    /// </summary>
    public abstract class BaseEntity
    {
        public Guid Id { get; set; }
        public DateTime CreatedAt { get; set; }
    }
}
