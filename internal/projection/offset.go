package projection

import "gorm.io/gorm"

// Offset tracks the last processed Kafka offset per topic+partition.
// Stored in the same DB as projections for atomic updates.
type Offset struct {
	ID        uint   `gorm:"primaryKey"`
	Topic     string `gorm:"uniqueIndex:idx_topic_partition"`
	Partition int32  `gorm:"uniqueIndex:idx_topic_partition"`
	Offset    int64
}

// SaveOffset upserts the offset within an existing GORM transaction.
func SaveOffset(tx *gorm.DB, topic string, partition int32, offset int64) error {
	var existing Offset
	result := tx.Where("topic = ? AND partition = ?", topic, partition).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		return tx.Create(&Offset{
			Topic:     topic,
			Partition: partition,
			Offset:    offset,
		}).Error
	}

	if result.Error != nil {
		return result.Error
	}

	// Only advance forward
	if offset > existing.Offset {
		return tx.Model(&existing).Update("offset", offset).Error
	}
	return nil
}
