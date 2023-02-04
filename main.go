package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

func main() {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("could not connect!! ", err)
		return
	}
	fmt.Println("Connected!")

	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal("disconnecting!! ", err)
		}
	}()
	db := client.Database("heatpump")

	if collections, err := db.ListCollectionNames(ctx, bson.D{}); err != nil {
		fmt.Println(err.Error())
	} else {
		if !Contains(collections, "config") {
			if err := db.CreateCollection(ctx, "config"); err != nil {
				log.Fatal("could not create collection!! ", err)
			}
		}
	}

	app.Use(cors.New())

	configs := &ConfigStore{db}

	app.Get("/config", func(c *fiber.Ctx) error {
		configList, err := configs.ReadMany()

		if err != nil {
			log.Fatal(err)
		}

		return c.JSON(configList)
	})

	app.Get("/config/:id", func(c *fiber.Ctx) error {
		if err != nil {
			log.Fatal(err)
		}

		config, err := configs.ReadOne(c.Params("id"))

		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Response().SetStatusCode(http.StatusNotFound)
				return c.SendString("record not found")
			}

			log.Fatal(err)
		}

		fmt.Println(config)

		return c.JSON(config)
	})

	app.Post("/config", func(c *fiber.Ctx) error {
		var conf Config
		// unmarshal json into struct
		json.Unmarshal(c.Body(), &conf)

		if conf.Id != "" {
			conf.Id = ""
		}

		fmt.Printf("creating new config named %v\n", conf.Name)

		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		newId, err := configs.Create(conf)

		if err != nil {
			log.Fatal(err)
		}

		return c.SendString(newId)
	})

	app.Post("/config/activate/:id", func(c *fiber.Ctx) error {
		if err := configs.Activate(c.Params("id")); err != nil {
			log.Fatal(err)
		}

		// restart firmware
		cmd := exec.Command("kill", "-2", "heatpump-controller")
		if err := cmd.Run(); err != nil {
			fmt.Println("failed to restart firmware")
		}
		return c.SendStatus(http.StatusAccepted)
	})

	app.Put("/config/:id", func(c *fiber.Ctx) error {
		conf, err := configs.ReadOne(c.Params("id"))
		if err != nil {
			log.Fatal(err)
		}

		// unmarshal json into struct
		json.Unmarshal(c.Body(), &conf)

		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := configs.Update(conf); err != nil {
			log.Fatal(err)
		}

		return c.SendStatus(http.StatusCreated)
	})

	app.Delete("/config/:id", func(c *fiber.Ctx) error {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := configs.Delete(c.Params("id")); err != nil {
			log.Fatal(err)
		}

		return c.SendStatus(http.StatusAccepted)
	})

	if err := app.Listen(":8080"); err != nil {
		log.Fatal("app failed to start", err)
	}
}
